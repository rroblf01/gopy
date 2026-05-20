from dataclasses import dataclass


@dataclass
class Box:
    name: str
    qty: int


def main() -> None:
    b = Box("widget", 3)
    v = vars(b)
    print(v["name"])
    print(v["qty"])
    # dir() returns more items in CPython (dunders, inherited), so only
    # verify the user-declared subset is present.
    names: list[str] = dir(b)
    print("name" in names)
    print("qty" in names)


if __name__ == "__main__":
    main()
