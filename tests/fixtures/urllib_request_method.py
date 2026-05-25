from urllib.request import Request


def main() -> None:
    r = Request("http://example.com", method="DELETE")
    # Add a header after construction.
    r.add_header("X-Custom", "ok")
    # No actual HTTP call here — the smoke test is that Request() with
    # the method= kwarg compiles and builds the object without errors.
    print("built")


if __name__ == "__main__":
    main()
