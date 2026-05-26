import signal


def main() -> None:
    prev = signal.alarm(0)
    print(prev)


if __name__ == "__main__":
    main()
