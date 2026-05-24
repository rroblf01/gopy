from dataclasses import dataclass, field


@dataclass
class Person:
    name: str
    tags: list[str] = field(default_factory=list)
    score: int = 0


def main() -> None:
    p = Person("alice")
    p.tags.append("admin")
    p.tags.append("user")
    print(p.name, p.tags, p.score)

    p2 = Person("bob", ["guest"])
    print(p2.name, p2.tags, p2.score)

    p3 = Person("carol", ["staff"], 99)
    print(p3.name, p3.tags, p3.score)

    # default factory creates fresh list each time
    a = Person("a")
    b = Person("b")
    a.tags.append("only-a")
    print(a.tags)
    print(b.tags)


if __name__ == "__main__":
    main()
