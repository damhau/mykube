from __future__ import annotations

import time
from collections import defaultdict

from fastapi import HTTPException, Request


class RateLimiter:
    def __init__(self, max_per_minute: int = 20) -> None:
        self.max_per_minute = max_per_minute
        self._hits: dict[str, list[float]] = defaultdict(list)

    async def __call__(self, request: Request) -> None:
        ip = request.client.host if request.client else "unknown"
        now = time.monotonic()
        cutoff = now - 60
        hits = [t for t in self._hits[ip] if t > cutoff]
        if len(hits) >= self.max_per_minute:
            raise HTTPException(status_code=429, detail="Too many requests")
        hits.append(now)
        self._hits[ip] = hits
