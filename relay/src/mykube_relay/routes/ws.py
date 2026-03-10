from __future__ import annotations

import asyncio
import logging

from fastapi import APIRouter, WebSocket, WebSocketDisconnect
from starlette.websockets import WebSocketState

from ..models import SessionStatus
from ..session_store import SessionStore

logger = logging.getLogger(__name__)


def create_ws_router(store: SessionStore) -> APIRouter:
    r = APIRouter(prefix="/ws")

    @r.websocket("/agent/{session_id}")
    async def agent_ws(websocket: WebSocket, session_id: str) -> None:
        session = await store.get_session(session_id)
        if session.agent_ws is not None:
            await websocket.close(code=4409, reason="agent already connected")
            return
        await websocket.accept()
        session.agent_ws = websocket

        try:
            # Poll until client connects
            while session.client_ws is None:
                await asyncio.sleep(0.05)

            # Notify agent that client is connected
            await websocket.send_text("paired")

            # Forward agent → client: read from agent WS, write to client WS
            client = session.client_ws
            while True:
                msg = await websocket.receive()
                msg_type = msg.get("type", "")
                if msg_type == "websocket.disconnect":
                    logger.info("Agent disconnected")
                    break
                if "text" in msg:
                    await client.send_text(msg["text"])
                elif "bytes" in msg:
                    await client.send_bytes(msg["bytes"])
        except (WebSocketDisconnect, RuntimeError) as e:
            logger.info("Agent handler exception: %s", e)
        finally:
            await _close_session(session, store, session_id)

    @r.websocket("/client/{session_id}")
    async def client_ws(websocket: WebSocket, session_id: str) -> None:
        session = await store.get_session(session_id)
        if session.status != SessionStatus.PAIRED:
            await websocket.close(code=4403, reason="session not paired")
            return
        if session.client_ws is not None:
            await websocket.close(code=4409, reason="client already connected")
            return

        await websocket.accept()
        session.client_ws = websocket

        try:
            # Wait until agent is ready (agent_ws set and paired signal sent)
            while session.agent_ws is None:
                await asyncio.sleep(0.05)

            # Forward client → agent: read from client WS, write to agent WS
            agent = session.agent_ws
            while True:
                msg = await websocket.receive()
                msg_type = msg.get("type", "")
                if msg_type == "websocket.disconnect":
                    logger.info("Client disconnected")
                    break
                if "text" in msg:
                    await agent.send_text(msg["text"])
                elif "bytes" in msg:
                    await agent.send_bytes(msg["bytes"])
        except (WebSocketDisconnect, RuntimeError) as e:
            logger.info("Client handler exception: %s", e)
        finally:
            await _close_session(session, store, session_id)

    return r


async def _close_session(session, store: SessionStore, session_id: str) -> None:
    for ws in (session.agent_ws, session.client_ws):
        if ws:
            try:
                if ws.client_state == WebSocketState.CONNECTED:
                    await ws.close()
            except Exception:
                pass
    await store.remove_session(session_id)
