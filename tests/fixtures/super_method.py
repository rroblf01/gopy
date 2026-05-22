class Animal:
    def __init__(self, name: str) -> None:
        self.name = name

    def describe(self) -> str:
        return "animal " + self.name


class Dog(Animal):
    def __init__(self, name: str, breed: str) -> None:
        super().__init__(name)
        self.breed = breed

    def describe(self) -> str:
        base: str = super().describe()
        return base + " (" + self.breed + ")"


def main() -> None:
    d = Dog("Rex", "lab")
    print(d.describe())


if __name__ == "__main__":
    main()
