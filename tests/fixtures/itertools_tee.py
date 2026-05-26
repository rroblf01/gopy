import itertools


def main() -> None:
    src = [1, 2, 3]
    parts = itertools.tee(src, 3)
    print(len(parts))


if __name__ == "__main__":
    main()
