from fastapi import APIRouter

from ..models import CreateSessionResponse, PairRequest, PairResponse
from ..session_store import SessionStore


def create_api_router(store: SessionStore) -> APIRouter:
    r = APIRouter(prefix="/api")

    @r.post("/sessions", response_model=CreateSessionResponse)
    async def create_session() -> CreateSessionResponse:
        session = await store.create_session()
        return CreateSessionResponse(session_id=session.session_id, code=session.code)

    @r.post("/pair", response_model=PairResponse)
    async def pair_session(body: PairRequest) -> PairResponse:
        session = await store.pair_session(body.code)
        return PairResponse(session_id=session.session_id)

    @r.get("/health")
    async def health() -> dict[str, str]:
        return {"status": "ok"}

    return r
