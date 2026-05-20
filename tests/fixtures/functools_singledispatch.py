from functools import singledispatch


@singledispatch
def to_int(v) -> int:
    return 0


def main() -> None:
    print(to_int(42))


if __name__ == "__main__":
    main()
