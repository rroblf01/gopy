def divmod_ints(a: int, b: int) -> tuple[int, int]:
    return a // b, a % b


def main() -> None:
    q, r = divmod_ints(17, 5)
    print(q)
    print(r)
    q, r = divmod_ints(20, 4)
    print(q)
    print(r)


if __name__ == "__main__":
    main()
