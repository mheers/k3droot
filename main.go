package main

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mheers/k3droot/helpers"
)

const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

var choice item

type item string

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s string) string {
			return selectedItemStyle.Render("> " + s)
		}
	}

	fmt.Fprintf(w, fn(str))
}

type model struct {
	list     list.Model
	items    []item
	choice   string
	quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i)
				choice = i
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.choice != "" {
		return ""
	}
	if m.quitting {
		return quitTextStyle.Render("No choice? No problem. \n\n")
	}
	return "\n" + m.list.View()
}

func main() {
	_, err := helpers.Init()
	if err != nil {
		panic(err)
	}

	k3d, err := helpers.IsK3d()
	if err != nil {
		panic(err)
	}
	if !k3d {
		fmt.Println("Not a k3d cluster")
		os.Exit(1)
	}

	var podName = ""

	// check if there is an argument
	if len(os.Args) >= 2 {
		// check if the argument is a pod name
		podName = os.Args[1]
	} else {
		pods, err := helpers.K8s.GetRunningPodsInCurrentNamespace()
		if err != nil {
			panic(err)
		}

		items := []list.Item{}
		for _, i := range pods {
			for _, c := range i.Spec.Containers {
				items = append(items, item(fmt.Sprintf("%s: %s", i.Name, c.Name)))
			}
		}

		const defaultWidth = 20

		l := list.New(items, itemDelegate{}, defaultWidth, listHeight)

		namespace := helpers.K8s.GetNamespace()
		title := fmt.Sprintf("There are %d running pods in %s:", len(pods), namespace)
		l.Title = title
		l.SetShowStatusBar(false)
		l.SetFilteringEnabled(false) // TODO: implement
		l.Styles.Title = titleStyle
		l.Styles.PaginationStyle = paginationStyle
		l.Styles.HelpStyle = helpStyle

		m := model{list: l}

		_, err = tea.NewProgram(m).StartReturningModel()
		if err != nil {
			fmt.Println("Error running program:", err)
			os.Exit(1)
		}

		if choice != "" {
			podName = string(choice)
		}
	}

	if podName == "" {
		fmt.Println("No pod name provided")
		os.Exit(1)
	}

	fmt.Printf("gaining root access into: %s\n", string(choice))
	err = helpers.RootIntoPodContainer(podName)
	if err != nil {
		panic(err)
	}
}
