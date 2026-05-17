class ValueError(Exception):
    def __init__(self, msg: str) -> None:
        super().__init__(msg)


def parse_age(s: str) -> int:
    if s == "bad":
        raise ValueError("bad input")
    return 42


def main() -> None:
    try:
        x: int = parse_age("ok")
        print(x)
        y: int = parse_age("bad")
        print(y)
    except ValueError as e:
        print("caught:")
        print(str(e))
    finally:
        print("done")


if __name__ == "__main__":
    main()
