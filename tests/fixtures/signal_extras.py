import signal


def main() -> None:
    sigs = signal.valid_signals()
    print(len(sigs) > 0)
    name = signal.strsignal(signal.SIGTERM)
    print(len(name) > 0)
    print(signal.ITIMER_REAL)


if __name__ == "__main__":
    main()
