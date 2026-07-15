package statusnotifier

import (
	"fmt"
	"os"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

const (
	itemInterface    = "org.kde.StatusNotifierItem"
	itemPath         = dbus.ObjectPath("/StatusNotifierItem")
	watcherService   = "org.kde.StatusNotifierWatcher"
	watcherPath      = dbus.ObjectPath("/StatusNotifierWatcher")
	watcherInterface = "org.kde.StatusNotifierWatcher"
)

// IconPixmap matches the StatusNotifierItem (width, height, ARGB data) tuple.
type IconPixmap struct {
	Width  int32
	Height int32
	Data   []byte
}

// ToolTip matches the StatusNotifierItem tooltip structure.
type ToolTip struct {
	IconName   string
	IconPixmap []IconPixmap
	Title      string
	Text       string
}

// Item is a native StatusNotifierItem hosted by the desktop session.
type Item struct {
	conn        *dbus.Conn
	properties  *prop.Properties
	serviceName string
}

// New creates and registers an InputScout hardware status item.
func New(title, iconName, description string) (*Item, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("connect to session bus: %w", err)
	}
	item := &Item{
		conn:        conn,
		serviceName: fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid()),
	}
	reply, err := conn.RequestName(item.serviceName, dbus.NameFlagDoNotQueue)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("request status notifier bus name: %w", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		conn.Close()
		return nil, fmt.Errorf("status notifier bus name is already owned: %s", reply)
	}

	properties := prop.Map{
		itemInterface: {
			"Category":            {Value: "Hardware", Emit: prop.EmitTrue},
			"Id":                  {Value: "inputscout", Emit: prop.EmitTrue},
			"Title":               {Value: title, Emit: prop.EmitTrue},
			"Status":              {Value: "Active", Emit: prop.EmitTrue},
			"WindowId":            {Value: int32(0), Emit: prop.EmitTrue},
			"IconName":            {Value: iconName, Emit: prop.EmitTrue},
			"IconPixmap":          {Value: []IconPixmap{}, Emit: prop.EmitTrue},
			"OverlayIconName":     {Value: "", Emit: prop.EmitTrue},
			"OverlayIconPixmap":   {Value: []IconPixmap{}, Emit: prop.EmitTrue},
			"AttentionIconName":   {Value: "battery-low", Emit: prop.EmitTrue},
			"AttentionIconPixmap": {Value: []IconPixmap{}, Emit: prop.EmitTrue},
			"AttentionMovieName":  {Value: "", Emit: prop.EmitTrue},
			"IconThemePath":       {Value: "", Emit: prop.EmitTrue},
			"ToolTip":             {Value: ToolTip{IconName: iconName, IconPixmap: []IconPixmap{}, Title: title, Text: description}, Emit: prop.EmitTrue},
			"ItemIsMenu":          {Value: false, Emit: prop.EmitTrue},
			"Menu":                {Value: dbus.ObjectPath("/NO_DBUSMENU"), Emit: prop.EmitTrue},
		},
	}
	item.properties, err = prop.Export(conn, itemPath, properties)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("export status notifier properties: %w", err)
	}
	if err := conn.Export(item, itemPath, itemInterface); err != nil {
		conn.Close()
		return nil, fmt.Errorf("export status notifier methods: %w", err)
	}
	itemIntrospection := introspect.Interface{
		Name:       itemInterface,
		Methods:    introspect.Methods(item),
		Properties: item.properties.Introspection(itemInterface),
		Signals: []introspect.Signal{
			{Name: "NewTitle"},
			{Name: "NewIcon"},
			{Name: "NewAttentionIcon"},
			{Name: "NewOverlayIcon"},
			{Name: "NewToolTip"},
			{Name: "NewStatus", Args: []introspect.Arg{{Name: "status", Type: "s"}}},
		},
	}
	node := &introspect.Node{
		Name: string(itemPath),
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			itemIntrospection,
		},
	}
	if err := conn.Export(introspect.NewIntrospectable(node), itemPath, introspect.IntrospectData.Name); err != nil {
		conn.Close()
		return nil, fmt.Errorf("export status notifier introspection: %w", err)
	}
	if err := item.register(); err != nil {
		conn.Close()
		return nil, err
	}
	item.watchForWatcherRestart()
	return item, nil
}

// Update changes the visible icon and tooltip and emits standard SNI signals.
func (i *Item) Update(title, iconName, description string) error {
	if i.properties.GetMust(itemInterface, "Title") != title {
		i.properties.SetMust(itemInterface, "Title", title)
		if err := i.conn.Emit(itemPath, itemInterface+".NewTitle"); err != nil {
			return err
		}
	}
	if i.properties.GetMust(itemInterface, "IconName") != iconName {
		i.properties.SetMust(itemInterface, "IconName", iconName)
		if err := i.conn.Emit(itemPath, itemInterface+".NewIcon"); err != nil {
			return err
		}
	}
	tooltip := ToolTip{IconName: iconName, IconPixmap: []IconPixmap{}, Title: title, Text: description}
	i.properties.SetMust(itemInterface, "ToolTip", tooltip)
	return i.conn.Emit(itemPath, itemInterface+".NewToolTip")
}

// Notify sends a standard desktop notification.
func (i *Item) Notify(summary, body, iconName string, urgency byte) error {
	hints := map[string]dbus.Variant{
		"category": dbus.MakeVariant("device"),
		"urgency":  dbus.MakeVariant(urgency),
	}
	var notificationID uint32
	call := i.conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications").Call(
		"org.freedesktop.Notifications.Notify",
		0,
		"InputScout",
		uint32(0),
		iconName,
		summary,
		body,
		[]string{},
		hints,
		int32(-1),
	)
	if call.Err != nil {
		return call.Err
	}
	return call.Store(&notificationID)
}

// Activate shows the current status as a desktop notification.
func (i *Item) Activate(_, _ int32) *dbus.Error {
	go i.notifyCurrent()
	return nil
}

// SecondaryActivate behaves like a normal activation.
func (i *Item) SecondaryActivate(_, _ int32) *dbus.Error {
	go i.notifyCurrent()
	return nil
}

// ContextMenu shows status because this minimal item intentionally has no menu.
func (i *Item) ContextMenu(_, _ int32) *dbus.Error {
	go i.notifyCurrent()
	return nil
}

// Scroll is accepted but intentionally has no action.
func (i *Item) Scroll(_ int32, _ string) *dbus.Error {
	return nil
}

func (i *Item) notifyCurrent() {
	title := i.properties.GetMust(itemInterface, "Title").(string)
	iconName := i.properties.GetMust(itemInterface, "IconName").(string)
	tooltip := i.properties.GetMust(itemInterface, "ToolTip").(ToolTip)
	_ = i.Notify(title, tooltip.Text, iconName, 1)
}

func (i *Item) register() error {
	call := i.conn.Object(watcherService, watcherPath).Call(
		watcherInterface+".RegisterStatusNotifierItem",
		0,
		i.serviceName,
	)
	if call.Err != nil {
		return fmt.Errorf("register status notifier item: %w", call.Err)
	}
	return nil
}

func (i *Item) watchForWatcherRestart() {
	if err := i.conn.AddMatchSignal(
		dbus.WithMatchObjectPath("/org/freedesktop/DBus"),
		dbus.WithMatchInterface("org.freedesktop.DBus"),
		dbus.WithMatchMember("NameOwnerChanged"),
		dbus.WithMatchArg(0, watcherService),
	); err != nil {
		return
	}
	signals := make(chan *dbus.Signal, 4)
	i.conn.Signal(signals)
	go func() {
		for signal := range signals {
			if len(signal.Body) != 3 {
				continue
			}
			newOwner, ok := signal.Body[2].(string)
			if ok && newOwner != "" {
				_ = i.register()
			}
		}
	}()
}

// Close unregisters the item by releasing its D-Bus connection.
func (i *Item) Close() error {
	return i.conn.Close()
}
