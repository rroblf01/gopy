def check(n: int) -> int:
    assert n > 0, f"got {n}"
    return n * 2


def main() -> None:
    print(check(5))
    try:
        check(-1)
    except Exception as e:
        print("caught:", e)


if __name__ == "__main__":
    main()
