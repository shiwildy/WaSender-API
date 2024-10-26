import requests
import json
import os
import base64

def test_senddoc():
    api_endpoint = "http://localhost:8080/sendimg"
    bearer_token = "your_secret_token"
    req_data = {
        "to": "yournumber",
        "caption": "Test document"
    }

    temp_file = "res.png"

    with open(temp_file, "rb") as f:
        file_data = f.read()
        encoded_data = base64.b64encode(file_data).decode("utf-8")

    req_data["image"] = encoded_data
    headers = {
        "Authorization": f"Bearer {bearer_token}",
        "Content-Type": "application/json"
    }

    response = requests.post(api_endpoint, headers=headers, json=req_data)
    print(response.text)

if __name__ == "__main__":
    test_senddoc()