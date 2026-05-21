from typing import Final, Annotated, Mapping, MutableSequence


PI: Final[float] = 3.14159


DEFAULT: Final[int] = 42


class Registry:
    def __init__(self) -> None:
        self.items: list[int] = []


def lookup(m: Mapping[str, int], key: str) -> int:
    return m.get(key, -1)


def push(xs: MutableSequence[int], v: int) -> int:
    xs.append(v)
    return len(xs)


def annotated_param(v: Annotated[int, "positive"]) -> int:
    return v * 2


def main() -> None:
    print(PI)
    r = Registry()
    r.items.append(5)
    r.items.append(7)
    print(len(r.items))
    print(DEFAULT)
    d: dict[str, int] = {"a": 1, "b": 2}
    print(lookup(d, "a"))
    print(lookup(d, "missing"))
    xs: list[int] = [10, 20]
    print(push(xs, 30))
    print(annotated_param(7))


if __name__ == "__main__":
    main()
