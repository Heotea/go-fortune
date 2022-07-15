/**
 * The MIT License (MIT)
 *
 * Copyright (c) 2022 chunghha(chunghha@users.noreply.github.com)
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */
package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// scrapeCmd represents the scrape command
var scrapeCmd = &cobra.Command{
	Use:   "scrape",
	Short: "scrape the fortune teller site",
	Long: `scrape the fortune teller site with a date given. For example,

	go-fortune scrape`,
	Run: func(cmd *cobra.Command, args []string) {
		main()
	},
}

func init() {
	rootCmd.AddCommand(scrapeCmd)
}

type model struct {
	textInput textinput.Model
	spinner   spinner.Model

	typing  bool
	loading bool
	err     error
	date    string
}

type gotFortune struct {
	err  error
	date string
}

func main() {
	t := textinput.NewModel()
	t.Focus()
	t.CharLimit = 8
	t.Width = 9
	t.SetValue(time.Now().Format("20060102"))

	s := spinner.NewModel()
	s.Spinner = spinner.Dot

	initialModel := model{
		textInput: t,
		spinner:   s,
		typing:    true,
	}

	err := tea.NewProgram(initialModel).Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}

func (m model) fetchFortune(date string) tea.Cmd {
	return func() tea.Msg {
		d, err := scrape(date)
		if err != nil {
			return gotFortune{err: err}
		}

		return gotFortune{date: d}
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.typing {
				query := strings.TrimSpace(m.textInput.Value())
				if query != "" {
					m.typing = false
					m.loading = true

					return m, tea.Batch(
						spinner.Tick,
						m.fetchFortune(query),
					)
				}
			}
		case "esc":
			if !m.typing && !m.loading {
				m.typing = true
				m.err = nil

				return m, nil
			}
		}

	case gotFortune:
		m.loading = false

		if err := msg.err; err != nil {
			m.err = err
			return m, nil
		}

		m.date = msg.date

		return m, nil
	}

	if m.typing {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)

		return m, cmd
	}

	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)

		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	if m.typing {
		return fmt.Sprintf("운세 볼 날짜를 입력 하세요.\n%s", m.textInput.View()) + "\n"
	}

	if m.loading {
		return fmt.Sprintf("%s 운세를 읽는 중... 기다려 주세요.", m.spinner.View())
	}

	if err := m.err; err != nil {
		return fmt.Sprintf("운세를 읽기에 실패 하였습니다. %v", err)
	}

	return "\n\n오늘의 운세를 출력 하였습니다. - " + m.date + "\n\n" +
		"(종료하려면 ctrl+c, 다른 날짜를 입력하려면 esc 키를 누르세요.)\n"
}

// too weak to be broken
func scrape(d string) (string, error) {
	res, err := http.Get(getUrl(d))

	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", err
	}

	fmt.Printf("\n(완료)\n\n")

	// div tag - class=article
	doc.Find("div").Each(func(i int, s *goquery.Selection) {
		text := s.Find("span").Text()

		// the article div is at 170th index
		if i == 170 {
			fmt.Printf("%s\n", text)
		}
	})

	return d, nil
}

func getUrl(date string) string {
	viper.SetConfigName("fortune_config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/go-workspace/bin")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// postfix in the config starts with '_' which is just to prevent
	// disappearing of leading zeros so take it out by splitting as a workaround.
	postfix := strings.Split(viper.GetString("url.postfix"), "_")[1]

	return viper.GetString("url.base") + viper.GetString("url.prefix") + date + postfix
}
