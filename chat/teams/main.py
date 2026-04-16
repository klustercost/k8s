import asyncio
import re
import json
import logging

from azure.identity import WorkloadIdentityCredential
from microsoft_teams.api import MessageActivity, TypingActivityInput
from microsoft_teams.apps import ActivityContext, App
from config import Config
from query_data import query
from adaptive_cards import make_donut_card

logging.basicConfig(level=logging.INFO)

config = Config('TENANT_ID', 'CLIENT_ID', 'BOT_TYPE', 'MCP_CLIENT_ADDRESS')

def create_token_factory():
    def get_token(scopes, tenant_id=None):
        credential = WorkloadIdentityCredential(client_id=config.CLIENT_ID)
        if isinstance(scopes, str):
            scopes_list = [scopes]
        else:
            scopes_list = scopes
        token = credential.get_token(*scopes_list)
        return token.token
    return get_token

app = App(
    token=create_token_factory() if config.BOT_TYPE == "UserAssignedMsi" else None
)

def response_from_ctx(ctx, imperative_query:str=None):
    logging.info(f"Handling from {ctx.connection_name}")
    json_response = query(
        config.MCP_CLIENT_ADDRESS, 
        ctx.conversation_ref.user.id if ctx.conversation_ref.user.id else ctx.conversation_ref.user.aad_object_id,  
        imperative_query if imperative_query else ctx.activity.text
    )
    logging.info(f"Natural response: {json_response['natural']}")
    logging.debug(f"Full answer from MCP: {json.dumps(json_response)}")    
    return json_response

@app.on_message_pattern(re.compile(r"What is the biggest increase in cost?"))
async def handle_greeting(ctx: ActivityContext[MessageActivity]) -> None:
    await ctx.reply(TypingActivityInput())
    response = response_from_ctx(ctx,"Top 3 namespaces with increased memory yesterday compared to previous day")
    await ctx.send(f"{response['natural']}")   
    await ctx.send(make_donut_card(response["raw"]))

@app.on_message
async def handle_message(ctx: ActivityContext[MessageActivity]):
    await ctx.reply(TypingActivityInput())
    response = response_from_ctx(ctx)
    await ctx.send(f"{response['natural']}")
    #TODO: This is a bit of a hack to determine if we should send a card or not, we should have a more robust way to determine this in the future
    if type(response["raw"]) == list and len(response["raw"]) > 3:
        await ctx.send(make_donut_card(response["raw"]))

if __name__ == "__main__":
    asyncio.run(app.start())
