import difflib


def main() -> None:
    sm = difflib.SequenceMatcher(None, "abcdef", "abxdef")
    r = sm.ratio()
    # Both Python and gopy: 2*5/12 = 0.833... Round to 2 places.
    rounded = int(r * 100)
    print(rounded)
    sm2 = difflib.SequenceMatcher(None, "", "")
    print(sm2.ratio())
    sm3 = difflib.SequenceMatcher(None, "hello", "")
    print(sm3.ratio())


if __name__ == "__main__":
    main()
