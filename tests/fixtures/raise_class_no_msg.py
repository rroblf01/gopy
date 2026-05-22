def fail() -> None:
    raise ValueError


def main() -> None:
    try:
        fail()
    except ValueError:
        print("caught")


if __name__ == "__main__":
    main()
