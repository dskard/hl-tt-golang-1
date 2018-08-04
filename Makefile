SRC=main.go
PROG=counter-ctrl

all:
	go get github.com/gorilla/mux
	go build -o ${PROG} -i main.go

run:
	./${PROG}

clean:

dist-clean: clean
	rm -f \
	    ./${PROG} \
	    *.log
