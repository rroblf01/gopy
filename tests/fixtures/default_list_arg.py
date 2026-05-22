def add_to(item: int, target: list[int] = []) -> list[int]:
    fresh: list[int] = list(target)
    fresh.append(item)
    return fresh


def main() -> None:
    a = add_to(1)
    b = add_to(2)
    c = add_to(3, [10, 20])
    print(a)
    print(b)
    print(c)


if __name__ == "__main__":
    main()
