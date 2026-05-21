from typing import get_type_hints, get_args, get_origin


def add(a: int, b: int) -> int:
    return a + b


def main() -> None:
    h = get_type_hints(add)
    print(len(h) >= 0)
    args = get_args("placeholder")
    print(len(args) == 0)
    origin = get_origin("placeholder")
    print(origin is None)


if __name__ == "__main__":
    main()
