import traceback


def main() -> None:
    frames = traceback.format_stack()
    # Both runtimes return a non-empty list; print only that the call
    # produced output (CPython lists Python frames, gopy emits Go runtime
    # stack — different content but same "has-some-stack" smoke).
    print(len(frames) > 0)


if __name__ == "__main__":
    main()
