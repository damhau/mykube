from __future__ import annotations

import asyncio
from contextlib import asynccontextmanager
from typing import AsyncIterator

from fastapi import FastAPI

from .config import settings
from .errors import (
    InvalidCode,
    MaxPairAttemptsExceeded,
    SessionAlreadyPaired,
    SessionExpired,
    SessionNotFound,
    invalid_code_handler,
    max_pair_attempts_handler,
    session_already_paired_handler,
    session_expired_handler,
    session_not_found_handler,
)
from .routes.api import create_api_router
from .routes.ws import create_ws_router
from .session_store import SessionStore

store = SessionStore()


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    async def _cleanup_loop() -> None:
        while True:
            await asyncio.sleep(settings.CLEANUP_INTERVAL)
            await store.cleanup_expired()

    task = asyncio.create_task(_cleanup_loop())
    try:
        yield
    finally:
        task.cancel()
        try:
            await task
        except asyncio.CancelledError:
            pass


app = FastAPI(lifespan=lifespan)

app.include_router(create_api_router(store))
app.include_router(create_ws_router(store))

app.add_exception_handler(SessionNotFound, session_not_found_handler)
app.add_exception_handler(SessionExpired, session_expired_handler)
app.add_exception_handler(InvalidCode, invalid_code_handler)
app.add_exception_handler(MaxPairAttemptsExceeded, max_pair_attempts_handler)
app.add_exception_handler(SessionAlreadyPaired, session_already_paired_handler)
