def add(a: int, b: int, c: int) -> int:
    return a + b + c


def main() -> None:
    # Inline list-literal spread expands to positional args at lower time.
    print(add(*[1, 2, 3]))
    print(add(*[10, 20, 30]))


if __name__ == "__main__":
    main()
