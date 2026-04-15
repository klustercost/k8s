from microsoft_teams.cards import AdaptiveCard, TextBlock, DonutChart, DonutChartData, ActionSet

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

def make_donut_card(data):
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
                data=donuts_from_items(data)
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