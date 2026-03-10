from __future__ import annotations

import asyncio
import secrets
import uuid
from datetime import datetime, timedelta, timezone

from .config import settings
from .errors import (
    InvalidCode,
    MaxPairAttemptsExceeded,
    SessionAlreadyPaired,
    SessionExpired,
    SessionNotFound,
)
from .models import Session, SessionStatus


class SessionStore:
    def __init__(self) -> None:
        self._sessions: dict[str, Session] = {}
        self._code_index: dict[str, str] = {}
        self._lock = asyncio.Lock()

    _CODE_CHARS = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

    def _generate_code(self) -> str:
        return "".join(secrets.choice(self._CODE_CHARS) for _ in range(8))

    async def create_session(self) -> Session:
        async with self._lock:
            session_id = str(uuid.uuid4())
            code = self._generate_code()
            while code in self._code_index:
                code = self._generate_code()

            now = datetime.now(timezone.utc)
            session = Session(
                session_id=session_id,
                code=code,
                created_at=now,
                expires_at=now + timedelta(seconds=settings.SESSION_TTL_WAITING),
            )
            self._sessions[session_id] = session
            self._code_index[code] = session_id
            return session

    async def pair_session(self, code: str) -> Session:
        async with self._lock:
            session_id = self._code_index.get(code)
            if session_id is None:
                raise InvalidCode()

            session = self._sessions.get(session_id)
            if session is None:
                del self._code_index[code]
                raise SessionNotFound()

            now = datetime.now(timezone.utc)
            if now >= session.expires_at:
                del self._code_index[code]
                del self._sessions[session_id]
                raise SessionExpired()

            if session.status == SessionStatus.PAIRED:
                raise SessionAlreadyPaired()

            session.pair_attempts += 1
            if session.pair_attempts > settings.MAX_PAIR_ATTEMPTS:
                del self._code_index[code]
                del self._sessions[session_id]
                raise MaxPairAttemptsExceeded()

            session.status = SessionStatus.PAIRED
            session.paired_at = now
            session.expires_at = now + timedelta(seconds=settings.SESSION_TTL_PAIRED)
            del self._code_index[code]
            return session

    async def get_session(self, session_id: str) -> Session:
        async with self._lock:
            session = self._sessions.get(session_id)
            if session is None:
                raise SessionNotFound()

            now = datetime.now(timezone.utc)
            if now >= session.expires_at:
                self._code_index.pop(session.code, None)
                del self._sessions[session_id]
                raise SessionExpired()

            return session

    async def remove_session(self, session_id: str) -> None:
        async with self._lock:
            session = self._sessions.pop(session_id, None)
            if session is not None:
                self._code_index.pop(session.code, None)

    async def cleanup_expired(self) -> None:
        async with self._lock:
            now = datetime.now(timezone.utc)
            expired = [
                sid
                for sid, s in self._sessions.items()
                if now >= s.expires_at
            ]
            for sid in expired:
                session = self._sessions.pop(sid)
                self._code_index.pop(session.code, None)
