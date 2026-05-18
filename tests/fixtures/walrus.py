def compute() -> int:
    return 42


def main() -> None:
    # Walrus in if-condition: `n` is hoisted to a preceding assign so
    # both runtimes see it bound when the body runs.
    if (n := compute()) > 0:
        print(n)
        print(n * 2)
    # Walrus in while-condition: re-evaluated each iteration.
    i: int = 0
    total: int = 0
    while (k := i * i) < 25:
        total += k
        i += 1
    print(total)


if __name__ == "__main__":
    main()
