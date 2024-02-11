#https://learn.microsoft.com/en-us/rest/api/cost-management/retail-prices/azure-retail-prices

import requests
import json

class price_api:
    base_api_url = "https://prices.azure.com/api/retail/prices"

    def __init__(self, version="api-version=2021-10-01-preview"):
        self.api_version = version

    def __form_api(self):
        return self.base_api_url + "?" + self.api_version

    def __structure_data(self,os):
        #table_data = [['SKU', 'Retail Price', 'Unit of Measure', 'Region', 'Meter', 'Product Name']]
        table_data = []
        for item in self.json_data['Items']:
            if os.lower() == "linux" and 'windows' in item['productName'].lower():
                continue
            if os.lower() == "windows" and not 'windows' in item['productName'].lower():
                continue            
            meter = item['meterName']
            table_data.append([item['armSkuName'], item['retailPrice'], item['unitOfMeasure'], item['armRegionName'], meter, item['productName']])
        return table_data

    def query(self, region, sku, os):
        if os.lower() == "windows":
            filter = "and contains(productName, 'Windows')"
        else:
            filter = "and not contains(productName, 'Windows')"
        response = requests.get(self.__form_api(),params={'$filter': f"armRegionName eq '{region}' and armSkuName eq '{sku}' and priceType eq 'Consumption' and contains(meterName, 'Spot')"})
        self.json_data = json.loads(response.text)
        return self.__structure_data(os)