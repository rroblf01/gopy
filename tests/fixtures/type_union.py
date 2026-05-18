def describe(v: int | str) -> str:
    if isinstance(v, int):
        return "int"
    return "str"


def main() -> None:
    print(describe(5))
    print(describe("hi"))


if __name__ == "__main__":
    main()
