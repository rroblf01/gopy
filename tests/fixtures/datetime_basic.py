from datetime import datetime


def main() -> None:
    # Comparing the actual `now()` output would be flaky across the two
    # runtimes. We just verify that calling it returns a non-empty string-
    # representable value in both. Each transpiled call to datetime.now()
    # produces a `str` already (Go side uses a string-returning shim).
    s = str(datetime.now())
    print(len(s) > 0)


if __name__ == "__main__":
    main()
