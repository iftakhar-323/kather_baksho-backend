FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy pre-compiled binary
COPY community_care_service /app/

# Create directory for persistent SQLite DB volume
RUN mkdir -p /app/data

# Environment defaults
ENV PORT=8088
ENV DB_PATH=/app/data/kather_baksho.db
ENV GIN_MODE=release

EXPOSE 8088

# Command to run microservice
CMD ["/app/community_care_service"]

