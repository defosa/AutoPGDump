# AutoPGDump
AutoPGDump is a Go program that automates the process of creating PostgreSQL database backups.

AutoPGDump is a program written in Go that automates the process of creating backups for a PostgreSQL database. The program uses the pg_dump utility to create consistent backups of the database, even if it is being used concurrently. The backups are saved in gzip format and uploaded to an S3 bucket. <br>The program is designed to be run in Kubernetes.
To use AutoPGDump, the user must set environment variables for the database name, username, password, host, and port.<br> The program will then execute the pg_dump command to create a backup of the database. The program uses a ticker to send a signal every 4 hours to create new backups. If the size of the backup file is 0, the program will issue an error.


<br>The program can be useful for those who want to create a simple and efficient solution for backing up a PostgreSQL database. The program can be run in Kubernetes and the backups can be saved in an S3 bucket for added security.<br> The program can be customized to suit the user's needs.
