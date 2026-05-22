import fcntl


def main() -> None:
    print(fcntl.LOCK_SH)
    print(fcntl.LOCK_EX)
    print(fcntl.LOCK_NB)
    print(fcntl.LOCK_UN)
    print(fcntl.F_GETFD)
    print(fcntl.F_SETFD)


if __name__ == "__main__":
    main()
