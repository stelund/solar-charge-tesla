import requests
import random
import string
import hashlib
import base64
import re
import sys
from urllib.parse import urlencode, parse_qs


def main(identity, credentials):
    state = "".join(
        [random.choice(string.ascii_letters + "0123456789") for x in range(20)]
    )
    code_verifier = "".join(
        [random.choice(string.ascii_letters + "0123456789") for x in range(86)]
    ).encode("ascii")
    code_challenge = base64.urlsafe_b64encode(hashlib.sha256(code_verifier).digest())
    session = requests.Session()

    params = {
        "client_id": "ownerapi",
        "code_challenge": code_challenge,
        "code_challenge_method": "S256",
        "redirect_uri": "https://auth.tesla.com/void/callback",
        "response_type": "code",
        "scope": "openid email offline_access",
        "state": state,
    }

    r = session.get("https://auth.tesla.com/oauth2/v3/authorize?" + urlencode(params))
    r.raise_for_status()
    form = dict(re.findall('input.*type="hidden".*name="(.*)".*value="(.*?)"', r.text))
    form["identity"] = identity
    form["credential"] = credentials

    r = session.post(
        "https://auth.tesla.com/oauth2/v3/authorize?" + urlencode(params),
        data=form,
        allow_redirects=False,
    )
    r.raise_for_status()
    url = r.headers["Location"]
    data = parse_qs(url.split("?")[1])

    token_data = {
        "grant_type": "authorization_code",
        "client_id": "ownerapi",
        "code": data["code"],
        "code_verifier": code_verifier,
        "redirect_uri": "https://auth.tesla.com/void/callback",
    }
    r = session.post("https://auth.tesla.com/oauth2/v3/token", token_data)
    r.raise_for_status()
    print(f"access_token is {r.json()['access_token']}")
    print(f"refresh_token is {r.json()['refresh_token']}")


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print('usage: tesla_login.py <username> <password>')
        sys.exit(1)
    main(sys.argv[1], sys.argv[2])