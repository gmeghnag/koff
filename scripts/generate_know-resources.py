import sys
def generate_yaml(input_file):
    bindings = {}
    with open(input_file, 'r') as file:
        lines = file.readlines()

        for line in lines:
            columns = line.strip().split()

            if len(columns) == 5:
                key = columns[4].lower()
                value = {
                    "group": columns[2].split("/")[0] if columns[2] !="v1" else "core",
                    "name": columns[4].lower(),
                    "namespaced": columns[3],
                    "plural": columns[0]
                }
                bindings[key] = value
                key = columns[0].lower()
                value = {
                    "group": columns[2].split("/")[0] if columns[2] !="v1" else "core",
                    "name": columns[4].lower(),
                    "namespaced": columns[3],
                    "plural": columns[0]
                }
                bindings[key] = value
                for alias in columns[1].split(","):
                    key = alias
                    value = {
                        "group": columns[2].split("/")[0] if columns[2] !="v1" else "core",
                        "name": columns[4].lower(),
                        "namespaced": columns[3],
                        "plural": columns[0]
                    }
                    bindings[key] = value
            if len(columns) == 4:
                key = columns[3].lower()
                value = {
                    "group": columns[1].split("/")[0] if columns[1] !="v1" else "core",
                    "name": columns[3].lower(),
                    "namespaced": columns[2],
                    "plural": columns[0]
                }
                bindings[key] = value
                key = columns[0].lower()
                value = {
                    "group": columns[1].split("/")[0] if columns[1] !="v1" else "core",
                    "name": columns[3].lower(),
                    "namespaced": columns[2],
                    "plural": columns[0]
                }
                bindings[key] = value
    yaml_output = ""
    for key, value in bindings.items():
        yaml_output += f"{key}:\n"
        for k, v in value.items():
            yaml_output += f"  {k}: {v}\n"

    return yaml_output


input_file = sys.argv[1]

output = generate_yaml(input_file)
print(output)