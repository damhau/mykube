from fastapi import Request
from fastapi.responses import JSONResponse


class SessionNotFound(Exception):
    pass


class SessionExpired(Exception):
    pass


class InvalidCode(Exception):
    pass


class MaxPairAttemptsExceeded(Exception):
    pass


class SessionAlreadyPaired(Exception):
    pass


async def session_not_found_handler(request: Request, exc: SessionNotFound) -> JSONResponse:
    return JSONResponse(status_code=404, content={"error": "session not found"})


async def session_expired_handler(request: Request, exc: SessionExpired) -> JSONResponse:
    return JSONResponse(status_code=410, content={"error": "session expired"})


async def invalid_code_handler(request: Request, exc: InvalidCode) -> JSONResponse:
    return JSONResponse(status_code=400, content={"error": "invalid code"})


async def max_pair_attempts_handler(request: Request, exc: MaxPairAttemptsExceeded) -> JSONResponse:
    return JSONResponse(status_code=429, content={"error": "max pair attempts exceeded"})


async def session_already_paired_handler(request: Request, exc: SessionAlreadyPaired) -> JSONResponse:
    return JSONResponse(status_code=409, content={"error": "session already paired"})
