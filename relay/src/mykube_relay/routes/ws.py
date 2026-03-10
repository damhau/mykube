from __future__ import annotations

import asyncio

from fastapi import APIRouter, WebSocket, WebSocketDisconnect

from ..models import SessionStatus
from ..session_store import SessionStore


def create_ws_router(store: SessionStore) -> APIRouter:
    r = APIRouter(prefix="/ws")

    async def _forward(src: WebSocket, dst: WebSocket) -> None:
        try:
            while True:
                data = await src.receive_text()
                await dst.send_text(data)
        except WebSocketDisconnect:
            pass

    @r.websocket("/agent/{session_id}")
    async def agent_ws(websocket: WebSocket, session_id: str) -> None:
        session = await store.get_session(session_id)
        await websocket.accept()
        session.agent_ws = websocket

        try:
            # Poll until client connects
            while session.client_ws is None:
                await asyncio.sleep(0.05)

            # Both sides connected - bridge messages
            client = session.client_ws
            t1 = asyncio.create_task(_forward(websocket, client))
            t2 = asyncio.create_task(_forward(client, websocket))
            done, pending = await asyncio.wait(
                [t1, t2], return_when=asyncio.FIRST_COMPLETED
            )
            for t in pending:
                t.cancel()
        except WebSocketDisconnect:
            pass
        finally:
            try:
                await websocket.close()
            except Exception:
                pass
            if session.client_ws:
                try:
                    await session.client_ws.close()
                except Exception:
                    pass
            await store.remove_session(session_id)

    @r.websocket("/client/{session_id}")
    async def client_ws(websocket: WebSocket, session_id: str) -> None:
        session = await store.get_session(session_id)
        if session.status != SessionStatus.PAIRED:
            await websocket.close(code=4403, reason="session not paired")
            return

        await websocket.accept()
        session.client_ws = websocket

        try:
            # Keep connection alive; bridging is driven by agent handler
            while True:
                await asyncio.sleep(1)
        except asyncio.CancelledError:
            pass
        finally:
            try:
                await websocket.close()
            except Exception:
                pass
            if session.agent_ws:
                try:
                    await session.agent_ws.close()
                except Exception:
                    pass
            await store.remove_session(session_id)

    return r
