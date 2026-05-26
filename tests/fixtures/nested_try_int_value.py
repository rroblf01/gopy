def main() -> None:
    def parse(s: str) -> int:
        try:
            return int(s)
        except ValueError:
            return -1
        except Exception:
            return -2

    print(parse("42"))
    print(parse("nope"))


if __name__ == "__main__":
    main()
