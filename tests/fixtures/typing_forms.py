from typing import Optional, Union, List, Dict


def first_or_fallback(xs: List[int], fallback: Optional[int]) -> int:
    if len(xs) > 0:
        return xs[0]
    if fallback is None:
        return -1
    return int(fallback)


def describe(v: Union[int, str]) -> str:
    if isinstance(v, int):
        return "int"
    return "str"


def main() -> None:
    print(first_or_fallback([10, 20], 99))
    print(first_or_fallback([], 7))
    print(first_or_fallback([], None))
    print(describe(5))
    print(describe("hi"))
    d: Dict[str, int] = {"a": 1, "b": 2}
    print(d["a"])


if __name__ == "__main__":
    main()
