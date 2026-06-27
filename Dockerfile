FROM heroiclabs/nakama-pluginbuilder:3.26.0 AS builder

ENV GO111MODULE=on
WORKDIR /backend
COPY modules/ ./
RUN go mod tidy && go build --trimpath --buildmode=plugin -o ./backend.so

FROM scratch
COPY --from=builder /backend/backend.so /backend.so
