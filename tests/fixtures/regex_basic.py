import re


def main() -> None:
    text: str = "ada 36 grace 47 alan 41"
    # findall and sub return identical types in CPython and the gopy shim,
    # so we exercise just those two for cross-runtime parity. search/match
    # diverge (Match object vs string) and require a thin user shim if you
    # need them in both runtimes today.
    nums: list[str] = re.findall("[0-9]+", text)
    print(len(nums))
    for n in nums:
        print(n)
    rep: str = re.sub("[0-9]+", "N", text)
    print(rep)
    words: list[str] = re.findall("[a-z]+", text)
    print(len(words))
    for w in words:
        print(w)


if __name__ == "__main__":
    main()
