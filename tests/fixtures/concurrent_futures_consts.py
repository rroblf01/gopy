import concurrent.futures


def main() -> None:
    print(concurrent.futures.FIRST_COMPLETED)
    print(concurrent.futures.ALL_COMPLETED)
    print(concurrent.futures.FIRST_EXCEPTION)


if __name__ == "__main__":
    main()
