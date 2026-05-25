import signal


def handler(signum: int, frame: object) -> None:
    print("got", signum)


def main() -> None:
    # Just register a handler; don't actually deliver a signal (testing
    # delivery requires platform-specific kill() + sleep loop which
    # makes the fixture flaky). The smoke test is that signal.signal()
    # accepts a callable for SIGTERM without error.
    signal.signal(signal.SIGTERM, handler)
    print("registered")


if __name__ == "__main__":
    main()
