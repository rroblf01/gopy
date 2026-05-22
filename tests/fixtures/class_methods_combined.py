class Person:
    def __init__(self, name: str, age: int) -> None:
        self.name = name
        self.age = age

    def older_by(self, years: int) -> "Person":
        return Person(self.name, self.age + years)

    @staticmethod
    def adult_age() -> int:
        return 18

    @classmethod
    def baby(cls, name: str) -> "Person":
        return cls(name, 0)


def main() -> None:
    p = Person("Alice", 25)
    q = p.older_by(5)
    print(q.name, q.age)
    print(Person.adult_age())
    b = Person.baby("Tim")
    print(b.name, b.age)


if __name__ == "__main__":
    main()
