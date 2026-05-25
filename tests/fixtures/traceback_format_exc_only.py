import traceback


def main() -> None:
    try:
        raise ValueError("bad thing")
    except ValueError as e:
        parts: list[str] = traceback.format_exception_only(type(e), e)
        for line in parts:
            # CPython appends a trailing newline; gopy does too. Strip
            # to keep the visible payload portable.
            print(line.strip())


if __name__ == "__main__":
    main()
