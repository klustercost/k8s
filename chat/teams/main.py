import asyncio
import re
import json
import logging

from azure.identity import WorkloadIdentityCredential
from microsoft_teams.api import MessageActivity, TypingActivityInput
from microsoft_teams.apps import ActivityContext, App
from microsoft_teams.cards import AdaptiveCard, TextBlock, DonutChart, DonutChartData, ActionSet
from config import Config
from query_data import query

logging.basicConfig(level=logging.INFO)

config = Config('TENANT_ID', 'CLIENT_ID', 'BOT_TYPE', 'MCP_CLIENT_ADDRESS')

def donuts_from_items(items):
    data = []
    key = None
    val = None
    for item in items:
        if key is None:
            keys = iter(item)
            key =  next(keys)
            val =  next(keys)
        data.append(DonutChartData(legend=item[key], value=item[val]))
    return data

def make_test_card(donut_data):
    return AdaptiveCard(
        schema="http://adaptivecards.io/schemas/adaptive-card.json",
        version="1.6",
        body=[
            TextBlock(
                text="The chart below shows this data",
                wrap=True
            ),
            DonutChart(
                title="Data Chart",
                data=donut_data
            ),
            ActionSet(actions=[
                {
                    "type": "Action.OpenUrl",
                    "url": "https://app.powerbi.com/groups/me/reports/65dad606-1ab5-4faa-b6fb-fd69e3751e4f/75c3436379060bbd7ab0",
                    "title": "More details",
                }
            ]),
        ]
    )

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

@app.on_message_pattern(re.compile(r"hello|hi|greetings"))
async def handle_greeting(ctx: ActivityContext[MessageActivity]) -> None:
    await ctx.send("Hello! How can I assist you today?")


@app.on_message
async def handle_message(ctx: ActivityContext[MessageActivity]):
    await ctx.reply(TypingActivityInput())
    response = json.loads(query(config.MCP_CLIENT_ADDRESS, ctx.activity.text))
    natural_response = response["natural"]
    logging.info(f"Handling from {ctx.connection_name} request {response} with answer {natural_response}")
    await ctx.send(f"{natural_response}")
    if type(response["raw"]) == list and len(response["raw"]) > 3:
        await ctx.send(make_test_card(donuts_from_items(response["raw"])))

if __name__ == "__main__":
    asyncio.run(app.start())
