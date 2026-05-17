from datetime import datetime, timedelta


def main() -> None:
    now1 = datetime.now()
    now2 = datetime.now()
    diff = now2 - now1
    # Both runtimes return a non-negative duration; we just verify the
    # arithmetic typechecks and produces a printable timedelta.
    print(str(diff) != "")
    later = now1 + timedelta(7)
    earlier = now1 - timedelta(1)
    # str() round-trips identically across CPython and the Go shim because
    # __Datetime.String() uses the same format.
    print(len(str(later)) > 0)
    print(len(str(earlier)) > 0)


if __name__ == "__main__":
    main()
