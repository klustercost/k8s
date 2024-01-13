#!/usr/bin/env python3

#https://learn.microsoft.com/en-us/rest/api/cost-management/retail-prices/azure-retail-prices

import requests
import json

class price_api:
    base_api_url = "https://prices.azure.com/api/retail/prices"

    def __init__(self, version="api-version=2021-10-01-preview"):
        self.api_version = version

    def __form_api(self):
        return self.base_api_url + "?" + self.api_version

    def __structure_data(self):
        table_data = [['SKU', 'Retail Price', 'Unit of Measure', 'Region', 'Meter', 'Product Name']]
        for item in self.json_data['Items']:
            meter = item['meterName']
            table_data.append([item['armSkuName'], item['retailPrice'], item['unitOfMeasure'], item['armRegionName'], meter, item['productName']])
        return table_data

    def query(self,query):
        response = requests.get(self.__form_api(),params={'$filter': query})
        self.json_data = json.loads(response.text)
        return self.__structure_data()