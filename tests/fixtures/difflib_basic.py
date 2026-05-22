import difflib


def main() -> None:
    cands: list[str] = ["apple", "application", "apply", "banana"]
    matches: list[str] = difflib.get_close_matches("appl", cands)
    print(len(matches))


if __name__ == "__main__":
    main()
