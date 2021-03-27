package templatedir

// SyncErrors ...
type SyncErrors chan error

// Failed ...
func (errs SyncErrors) Failed() bool {
	select {
	case err := <-errs:
		select {
		case errs <- err:
			return true
		default:
			return true
		}
	default:
		return false
	}
}

// Close ...
func (errs SyncErrors) Close() error {
	var err error
	select {
	case err = <-errs:
	default:
	}

	close(errs)
	return err
}

// SetFailedOnErr ...
func (errs SyncErrors) SetFailedOnErr(err error) bool {
	if err != nil {
		select {
		case errs <- err:
		default:
		}
		return true
	}

	return false
}
