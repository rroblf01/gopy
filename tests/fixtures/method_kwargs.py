class HTTPClient:
    def __init__(self, base: str) -> None:
        self.base = base

    def request(self, path: str, method: str = "GET", timeout: int = 30) -> str:
        return f"{method} {self.base}{path} timeout={timeout}"


def main() -> None:
    c: HTTPClient = HTTPClient("https://example.com")
    print(c.request("/users"))
    print(c.request("/users", "POST"))
    print(c.request("/users", method="DELETE"))
    print(c.request("/users", timeout=5, method="PUT"))


if __name__ == "__main__":
    main()
