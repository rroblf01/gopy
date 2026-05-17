from gopy_django.models import Model, CharField, IntegerField, BooleanField


class User(Model):
    name = CharField(max_length=100)
    age = IntegerField()
    active = BooleanField(default=True)


def main() -> None:
    # Constructor + save round-trips through the manager's records list.
    u1 = User(name="ada", age=36, active=True)
    u1.save()
    u2 = User(name="grace", age=47, active=True)
    u2.save()
    u3 = User(name="alan", age=41, active=False)
    u3.save()

    # all() preserves insertion order in both runtimes.
    everyone = User.objects.all()
    print(len(everyone))
    for u in everyone:
        print(u.name)

    # filter(**kwargs) selects records where every kwarg matches.
    grace = User.objects.filter(name="grace")
    print(len(grace))
    print(grace[0].age)

    # get(**kwargs) returns the unique match.
    only_ada = User.objects.get(name="ada")
    print(only_ada.age)

    # create(**kwargs) constructs + saves in one step.
    bob = User.objects.create(name="bob", age=29, active=True)
    print(bob.name)
    print(len(User.objects.all()))


if __name__ == "__main__":
    main()
