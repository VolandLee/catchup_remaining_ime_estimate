package main
import (
	"encoding/gob"
	"fmt"
	"github.com/wal-g/tracelog"
	"github.com/wal-g/wal-g/internal"
	"github.com/wal-g/wal-g/internal/compression"
	"github.com/wal-g/wal-g/internal/ioextensions"
	"github.com/wal-g/wal-g/utility"
	"io"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)



// Функция для запуска команд с разным количеством передаваемых параметров
func execCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}


// Функция определяет время создания резервной копии
func getBackupCreationTime(backupName string) (time.Time, error) {
	output, err := execCommand("wal-g", "backup-list", "--detail")
	if err != nil {
		return time.Time{}, err
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, backupName) {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return time.Parse("2006-01-02T15:04:05Z", parts[2])
			}
		}
	}
	return time.Time{}, fmt.Errorf("backup %s not found", backupName)
}


// Функция для получения списка WAL файлов, созданных после резервной копии
func getWALFilesSince(backupTime time.Time) ([]string, error) {
	output, err := execCommand("wal-g", "wal-show")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(output, "\n")
	var walFiles []string
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			walTime, err := time.Parse("2006-01-02T15:04:05Z", parts[0])
			if err == nil && walTime.After(backupTime) {
				walFiles = append(walFiles, parts[1])
			}
		}
	}
	return walFiles, nil
}

// Функция для оценки среднего времени применения одного WAL файла
func estimateWALApplyTime() (time.Duration, error) {
	// Для упрощения процедуры зададим постоянное время обработки одного WAL файла. 
	// Для улучшения прогнозирующей способности процедуры можно использовать статистически средние данные обработки WAL файла, полученные из логов BD.
	return 10 * time.Second, nil
}

// Функция для установления соединения и сбора необходимой информации (метрик).
func establishConnection(destination string) (*gob.Decoder, *gob.Encoder, error) {
	writer, decoder, encoder := startSendConnection(destination)

	var control PgControlData
	err := decoder.Decode(&control)
	if err != nil {
		return nil, nil, err
	}
	tracelog.InfoLogger.Printf("Destination control file %v", control)
	info, _, err := GetPgServerInfo(true)
	if err != nil {
		return nil, nil, err
	}
	tracelog.InfoLogger.Printf("Our system id %v", *info.systemIdentifier)
	if *info.systemIdentifier != control.SystemIdentifier {
		return nil, nil, fmt.Errorf("system identifiers do not match")
	}
	if control.CurrentTimeline != info.Timeline {
		return nil, nil, fmt.Errorf("destination is on timeline %v, but we are on %v",
			control.CurrentTimeline, info.Timeline)
	}

	return decoder, encoder, nil
}


// Основная функция для расчета времени до завершения catchup
func estimateCatchupTime(backupName string, destination string) error {
	backupTime, err := getBackupCreationTime(backupName)
	if err != nil {
		return err
	}
	walFiles, err := getWALFilesSince(backupTime)
	if err != nil {
		return err
	}
	avgWALApplyTime, err := estimateWALApplyTime()
	if err != nil {
		return err
	}

	// Установавливаем соединение
	decoder, encoder, err := establishConnection(destination)
	if err != nil {
		return err
	}

	var fileList internal.BackupFileList
	err = decoder.Decode(&fileList)
	if err != nil {
		return err
	}
	tracelog.InfoLogger.Printf("Received file list of %v files", len(fileList))

	// Получить текущий LSN, тоесть запись WAL журналов для восстановления откуда нужно начинать отсчёт.
	info, runner, err := GetPgServerInfo(true)
	if err != nil {
		return err
	}
	_, lsnStr, _, err := runner.StartBackup("")
	if err != nil {
		return err
	}
	lsn, err := ParseLSN(lsnStr)
	if err != nil {
		return err
	}


  
	// Оценка общего времени для применения WAL файлов
	totalWALFiles := len(walFiles)
	totalCatchupTime := time.Duration(totalWALFiles) * avgWALApplyTime

	tracelog.InfoLogger.Printf("Estimated catchup time: %v\n", totalCatchupTime)
	fmt.Printf("Estimated catchup time: %v\n", totalCatchupTime)

	return nil
}

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Usage: %s <backup_name> <destination>\n", os.Args[0])
	}
	backupName := os.Args[1]
	destination := os.Args[2]
	if err := estimateCatchupTime(backupName, destination); err != nil {
		log.Fatalf("Error estimating catchup time: %v\n", err)
	}
}
