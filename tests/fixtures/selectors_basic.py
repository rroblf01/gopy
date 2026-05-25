import selectors


def main() -> None:
    s = selectors.DefaultSelector()
    s.register(1, selectors.EVENT_WRITE, "stdout")
    # Use a tight timeout so both CPython (real epoll) and gopy
    # (stub) return quickly.
    events = s.select(0.01)
    print(len(events) >= 0)
    s.unregister(1)
    s.close()


if __name__ == "__main__":
    main()
