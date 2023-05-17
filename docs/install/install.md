# Install

## Download the latest binary release :material-github:
---

=== "Darwin :material-apple:"
    ```
    curl -sL https://github.com/gmeghnag/koff/releases/latest/download/koff_Darwin_x86_64.tar.gz | tar xzf - koff
    chmod +x ./koff
    ```
=== "Linux :simple-linux:"
    ``` aml
    curl -sL https://github.com/gmeghnag/koff/releases/latest/download/koff_Linux_x86_64.tar.gz | tar xzf - koff
    chmod +x ./koff   
    ```
=== "Windows :fontawesome-brands-windows:"
    ``` 
    curl.exe -sL "https://github.com/gmeghnag/koff/releases/latest/download/koff_Windows_x86_64.zip" -o koff.zip 
    tar -xf koff.zip
    ./koff.exe 
    ```


## Via `go install` :fontawesome-brands-golang:
---
```
go install github.com/gmeghnag/koff
```

## Build from the source code
---
```
git clone https://github.com/gmeghnag/koff.git
cd koff/
go install github.com/gmeghnag/koff
```
