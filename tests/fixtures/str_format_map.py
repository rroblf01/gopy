def main() -> None:
    d: dict[str, int] = {"x": 10, "y": 20}
    print("{x} and {y}".format_map(d))

    e: dict[str, str] = {"who": "world"}
    print("hello, {who}!".format_map(e))


if __name__ == "__main__":
    main()
