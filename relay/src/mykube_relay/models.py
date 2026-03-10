from __future__ import annotations

import enum
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Optional

from fastapi import WebSocket
from pydantic import BaseModel


class SessionStatus(enum.Enum):
    WAITING = "waiting"
    PAIRED = "paired"


@dataclass
class Session:
    session_id: str
    code: str
    status: SessionStatus = SessionStatus.WAITING
    agent_ws: Optional[WebSocket] = field(default=None, repr=False)
    client_ws: Optional[WebSocket] = field(default=None, repr=False)
    created_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    paired_at: Optional[datetime] = None
    pair_attempts: int = 0
    expires_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))


class CreateSessionResponse(BaseModel):
    session_id: str
    code: str


class PairRequest(BaseModel):
    code: str


class PairResponse(BaseModel):
    session_id: str
