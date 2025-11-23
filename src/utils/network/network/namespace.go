package network

import (
	"errors"
	"runtime"

	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"
)

func switchToNamespace(nsName string) (func(), error) {
	// Lock the OS thread so we don't switch namespaces for other goroutines
	// in this process.
	runtime.LockOSThread()

	origNS, origNsErr := netns.Get()
	if origNsErr != nil {
		runtime.UnlockOSThread()
		return nil, origNsErr
	}

	targetNS, targetNsErr := netns.GetFromName(nsName)
	if targetNsErr != nil {
		origNS.Close()
		runtime.UnlockOSThread()
		return nil, targetNsErr
	}

	if setNsErr := netns.Set(targetNS); setNsErr != nil {
		origNS.Close()
		targetNS.Close()
		runtime.UnlockOSThread()
		return nil, setNsErr
	}

	targetNS.Close()

	return func() {
		defer runtime.UnlockOSThread()
		defer origNS.Close()
		// Best effort to restore namespace - log errors but don't panic
		if err := netns.Set(origNS); err != nil {
			_ = err // Ignore restoration errors - thread will be unlocked anyway
		}
	}, nil
}

func createNamespace(nsName string) (int, error) {
	nsHandle, err := netns.GetFromName(nsName)
	if err == nil {
		return int(nsHandle), nil
	}

	if !errors.Is(err, unix.ENOENT) {
		return int(netns.None()), err
	}

	// Lock thread to prevent namespace switching affecting other goroutines
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Save current namespace
	origNS, origNsErr := netns.Get()
	if origNsErr != nil {
		return int(netns.None()), origNsErr
	}
	defer origNS.Close()

	// Create new namespace (this switches to it)
	nsHandle, newErr := netns.NewNamed(nsName)
	if newErr != nil {
		return int(netns.None()), newErr
	}

	// Switch back to original namespace
	if setErr := netns.Set(origNS); setErr != nil {
		nsHandle.Close()
		return int(netns.None()), setErr
	}

	return int(nsHandle), nil
}

func deleteNamespace(nsName string) error {
	return netns.DeleteNamed(nsName)
}
