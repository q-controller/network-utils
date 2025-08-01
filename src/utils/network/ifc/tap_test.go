package ifc

import (
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TapLinkManagerMock for tap tests
type TapLinkManagerMock struct {
	mock.Mock
}

func (m *TapLinkManagerMock) Exists(name string) (bool, error) {
	args := m.Called(name)
	return args.Bool(0), args.Error(1)
}
func (m *TapLinkManagerMock) SetMaster(name string, masterName string) error {
	return m.Called(name, masterName).Error(0)
}
func (m *TapLinkManagerMock) AddLink(name string, typ LinkType) error {
	return m.Called(name, typ).Error(0)
}
func (m *TapLinkManagerMock) SetIP(name string, ip net.IP, mask net.IPMask) error {
	return m.Called(name, ip, mask).Error(0)
}
func (m *TapLinkManagerMock) BringUp(name string) error {
	return m.Called(name).Error(0)
}
func (m *TapLinkManagerMock) DeleteLink(name string) error {
	return m.Called(name).Error(0)
}
func (m *TapLinkManagerMock) HasIP(name string, ip net.IP, mask net.IPMask) (bool, error) {
	args := m.Called(name, ip, mask)
	return args.Bool(0), args.Error(1)
}
func (m *TapLinkManagerMock) DisableTxOffloading(name string) error {
	return m.Called(name).Error(0)
}

func TestCreateTapWithManager_Success_NewTap(t *testing.T) {
	mgr := &TapLinkManagerMock{}
	mgr.On("Exists", "tap0").Return(false, nil)
	mgr.On("AddLink", "tap0", LinkTypeTap).Return(nil)
	mgr.On("SetMaster", "tap0", "br0").Return(nil)
	mgr.On("BringUp", "tap0").Return(nil)
	err := CreateTapWithManager(mgr, "tap0", "br0")
	require.NoError(t, err)
}

func TestCreateTapWithManager_Success_ExistingTap(t *testing.T) {
	mgr := &TapLinkManagerMock{}
	mgr.On("Exists", "tap0").Return(true, nil)
	mgr.On("BringUp", "tap0").Return(nil)
	err := CreateTapWithManager(mgr, "tap0", "br0")
	require.NoError(t, err)
}

func TestCreateTapWithManager_ExistsError(t *testing.T) {
	mgr := &TapLinkManagerMock{}
	mgr.On("Exists", "tap0").Return(false, errors.New("exists error"))
	err := CreateTapWithManager(mgr, "tap0", "br0")
	require.EqualError(t, err, "unexpected error checking link: exists error")
}

func TestCreateTapWithManager_AddLinkError(t *testing.T) {
	mgr := &TapLinkManagerMock{}
	mgr.On("Exists", "tap0").Return(false, nil)
	mgr.On("AddLink", "tap0", LinkTypeTap).Return(errors.New("add link failed"))
	err := CreateTapWithManager(mgr, "tap0", "br0")
	require.EqualError(t, err, "failed to add tap device tap0: add link failed")
}

func TestCreateTapWithManager_SetMasterError_DeleteSuccess(t *testing.T) {
	mgr := &TapLinkManagerMock{}
	mgr.On("Exists", "tap0").Return(false, nil)
	mgr.On("AddLink", "tap0", LinkTypeTap).Return(nil)
	mgr.On("SetMaster", "tap0", "br0").Return(errors.New("set master failed"))
	mgr.On("DeleteLink", "tap0").Return(nil)
	err := CreateTapWithManager(mgr, "tap0", "br0")
	require.EqualError(t, err, "failed to set master for tap tap0: set master failed")
}

func TestCreateTapWithManager_SetMasterError_DeleteError(t *testing.T) {
	mgr := &TapLinkManagerMock{}
	mgr.On("Exists", "tap0").Return(false, nil)
	mgr.On("AddLink", "tap0", LinkTypeTap).Return(nil)
	mgr.On("SetMaster", "tap0", "br0").Return(errors.New("set master failed"))
	mgr.On("DeleteLink", "tap0").Return(errors.New("delete failed"))
	err := CreateTapWithManager(mgr, "tap0", "br0")
	require.EqualError(t, err, "failed to set master for tap tap0: set master failed, failed to delete tap: delete failed")
}

func TestCreateTapWithManager_BringUpError_DeleteSuccess(t *testing.T) {
	mgr := &TapLinkManagerMock{}
	mgr.On("Exists", "tap0").Return(true, nil)
	mgr.On("BringUp", "tap0").Return(errors.New("bring up failed"))
	mgr.On("DeleteLink", "tap0").Return(nil)
	err := CreateTapWithManager(mgr, "tap0", "br0")
	require.EqualError(t, err, "failed to bring tap tap0 up: bring up failed")
}

func TestCreateTapWithManager_BringUpError_DeleteError(t *testing.T) {
	mgr := &TapLinkManagerMock{}
	mgr.On("Exists", "tap0").Return(true, nil)
	mgr.On("BringUp", "tap0").Return(errors.New("bring up failed"))
	mgr.On("DeleteLink", "tap0").Return(errors.New("delete failed"))
	err := CreateTapWithManager(mgr, "tap0", "br0")
	require.EqualError(t, err, "failed to bring tap tap0 up: bring up failed, failed to delete tap: delete failed")
}
