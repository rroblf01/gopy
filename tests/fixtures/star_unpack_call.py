def add3(a: int, b: int, c: int) -> int:
    return a + b + c


def label(prefix: str, name: str) -> str:
    return prefix + ":" + name


def main() -> None:
    t = [10, 20, 30]
    print(add3(*t))

    pair = ["hello", "ana"]
    print(label(*pair))


if __name__ == "__main__":
    main()
