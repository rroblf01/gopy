import select


def main() -> None:
    # CPython needs real fds; gopy stub accepts any. Use empty lists so
    # both paths agree on shape: 3-tuple of empty subsets.
    res = select.select([], [], [], 0)
    print(len(res))


if __name__ == "__main__":
    main()
