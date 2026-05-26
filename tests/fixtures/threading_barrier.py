import threading


def main() -> None:
    b = threading.Barrier(1)
    # Single-party barrier releases immediately. Verifies the .wait/.reset
    # / .parties API surface compiles and runs through.
    b.wait()
    b.reset()
    print(b.parties)
    print(b.broken)
    b.wait()
    print("ok")


if __name__ == "__main__":
    main()
