from dataclasses import dataclass, field


@dataclass
class Bag:
    name: str
    items: list[int] = field(default_factory=list)
    tags: dict[str, int] = field(default_factory=dict)


def main() -> None:
    a = Bag("a")
    a.items.append(1)
    a.items.append(2)
    a.tags["x"] = 7
    b = Bag("b")
    # Mutating a must not leak to b (defaults are fresh per instance).
    print(len(a.items))
    print(len(b.items))
    print(a.tags["x"])
    print(len(b.tags))


if __name__ == "__main__":
    main()
