/*
 * Copyright (c) 2016-2018 Readium Foundation
 *
 * Redistribution and use in source and binary forms, with or without modification,
 * are permitted provided that the following conditions are met:
 *
 *  1. Redistributions of source code must retain the above copyright notice, this
 *     list of conditions and the following disclaimer.
 *  2. Redistributions in binary form must reproduce the above copyright notice,
 *     this list of conditions and the following disclaimer in the documentation and/or
 *     other materials provided with the distribution.
 *  3. Neither the name of the organization nor the names of its contributors may be
 *     used to endorse or promote products derived from this software without specific
 *     prior written permission
 *
 *  THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 *  ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 *  WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 *  DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
 *  ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 *  (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 *  LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 *  ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 *  (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 *  SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package helpers

import (
	"fmt"
	got "html/template"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// FORMS

// These should probably use templates from or from lib, so that users can change what form fields get generated
// and use templ rather than fmt.Sprintf

// We need to set this token in the session on the get request for the form

// CSRF generates an input field tag containing a CSRF token
func CSRF() got.HTML {
	token := "my_csrf_token" // instead of generating this here, should we instead get router or app to generate and put into the context?
	output := fmt.Sprintf("<input type='hidden' name='csrf' value='%s'>", token)
	return got.HTML(output)
}

// Field accepts name string, value interface{}, fieldType string, args ...string
func Field(label string, name string, v interface{}, args ...string) got.HTML {
	attributes := ""
	if len(args) > 0 {
		attributes = strings.Join(args, " ")
	}
	// If no type, add it to attributes
	if !strings.Contains(attributes, "type=") {
		attributes = attributes + " type=\"text\""
	}
	tmpl := FIELD
	if label == "" {
		tmpl = FIELD_NO_LABEL
	}
	output := fmt.Sprintf(tmpl, Escape(label), Escape(name), Escape(fmt.Sprintf("%v", v)), attributes)
	return got.HTML(output)
}

// DateField sets up a date field with a data-date attribute storing the real date
func DateField(label string, name string, t time.Time, args ...string) got.HTML {
	// NB we use text type for date fields because of inconsistent browser behaviour
	// and to support our own date picker popups
	tmpl := DATE_FIELD
	attributes := ""
	if len(args) > 0 {
		attributes = strings.Join(args, " ")
	}
	output := fmt.Sprintf(tmpl, Escape(label), Escape(name), Escape(name), Date(t), Date(t, "2006-01-02"), attributes)
	return got.HTML(output)
}

// TextArea returns a field div containing a textarea
func TextArea(label string, name string, v interface{}, args ...string) got.HTML {
	attributes := ""
	if len(args) > 0 {
		attributes = strings.Join(args, " ")
	}
	fieldTemplate := TEXTAREA
	output := fmt.Sprintf(fieldTemplate,
		Escape(label),
		Escape(name),
		attributes, // NB we do not escape attributes, which may contain HTML
		v)          // NB value may contain HTML
	return got.HTML(output)
}

// TODO flip the select helpers to use Selectable all the time?
// Redefine concrete type Option as a Selectable and this should be doable?

// SelectName returns the public name for this select option
func (o SelectableOption) SelectName() string {
	return o.Name
}

// SelectValue returns the value for this select option
func (o SelectableOption) SelectValue() string {
	return o.Value
}

// StringOptions creates an array of selectables from strings
func StringOptions(args ...string) []SelectableOption {
	var options []SelectableOption
	// Construct a slice of options from these strings
	for _, s := range args {
		options = append(options, SelectableOption{s, s})
	}
	return options
}

// NumberOptions creates an array of selectables, with an optional min and max value supplied as arguments
func NumberOptions(args ...int64) []SelectableOption {
	min := int64(0)
	max := int64(50)
	if len(args) > 0 {
		min = args[0]
	}
	if len(args) > 1 {
		max = args[1]
	}
	var options []SelectableOption
	for i := min; i <= max; i++ {
		v := strconv.Itoa(int(i))
		n := v
		options = append(options, SelectableOption{n, v})
	}
	return options
}

// Better to use an interface and not reflect here - Would rather avoid use of reflect...

// OptionsForSelect creates a select field given an array of keys and values in order
func OptionsForSelect(value interface{}, options interface{}) got.HTML {
	stringValue := fmt.Sprintf("%v", value)
	output := ""
	switch reflect.TypeOf(options).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(options)
		for i := 0; i < s.Len(); i++ {
			o := s.Index(i).Interface().(Selectable)
			sel := ""
			if o.SelectValue() == stringValue {
				sel = "selected"
			}
			output += fmt.Sprintf(`<option value="%s" %s>%s</option>\n`, o.SelectValue(), sel, Escape(o.SelectName()))
		}
	}
	return got.HTML(output)
}

// SelectArray creates a select field given an array of keys and values in order
func SelectArray(label string, name string, value interface{}, options interface{}) got.HTML {
	stringValue := fmt.Sprintf("%v", value)
	tmpl := SELECT
	if label == "" {
		tmpl = SELECT_NO_LABEL
	}
	opts := ""
	switch reflect.TypeOf(options).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(options)
		for i := 0; i < s.Len(); i++ {
			o := s.Index(i).Interface().(Selectable)
			sel := ""
			if o.SelectValue() == stringValue {
				sel = "selected"
			}
			opts += fmt.Sprintf(`<option value="%s" %s>%s</option>\n`, o.SelectValue(), sel, Escape(o.SelectName()))
		}
	}
	output := fmt.Sprintf(tmpl, Escape(label), Escape(name), Escape(name), opts)
	return got.HTML(output)
}

// FIXME - make Option conform to Selectable interface and use that instead of concrete type below
// Select creates a select field given an array of keys and values in order
func Select(label string, name string, value int64, options []Option) got.HTML {
	tmpl := SELECT_NO_ID
	if label == "" {
		tmpl = SELECT_NO_ID_NO_LABEL
	}
	opts := ""
	for _, o := range options {
		s := ""
		if o.Id == value {
			s = "selected"
		}
		opts += fmt.Sprintf(`<option value="%d" %s>%s</option>`, o.Id, s, Escape(o.Name))
	}
	output := fmt.Sprintf(tmpl, Escape(label), Escape(name), "", opts)
	return got.HTML(output)
}

func DisabledSelect(label string, name string, value int64, options []Option) got.HTML {
	tmpl := SELECT_NO_ID
	if label == "" {
		tmpl = SELECT_NO_ID_NO_LABEL
	}
	opts := ""
	for _, o := range options {
		s := ""
		if o.Id == value {
			s = "selected"
		}
		opts += fmt.Sprintf(`<option value="%d" %s>%s</option>`, o.Id, s, Escape(o.Name))
	}
	output := fmt.Sprintf(tmpl, Escape(label), Escape(name), "disabled", opts)
	return got.HTML(output)
}

func GenericSelect(defaultOption string, name string, labelField string, valueField string, selectedValue int64, options interface{}) got.HTML {
	tmpl := SELECT
	opts := "<option value=\"0\">" + defaultOption + "</option>"
	switch reflect.TypeOf(options).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(options)
		for i := 0; i < s.Len(); i++ {
			element := s.Index(i)
			if element.Kind() == reflect.Ptr {
				element = element.Elem()
			}
			if element.FieldByName(labelField).Kind() != reflect.String {
				return got.HTML(fmt.Sprintf("Error : %s is NOT string or is UNKNOWN", labelField))

			}
			if element.FieldByName(valueField).Kind() != reflect.Int64 {
				return got.HTML(fmt.Sprintf("Error : %s is NOT int64 or is UNKNOWN", valueField))
			}
			currentOptionLabel := element.FieldByName(labelField).String()
			currentOptionValue := element.FieldByName(valueField).Int()
			sel := ""
			//fmt.Printf("'%s' = %d '%s' = %s %v\n", valueField, currentOptionValue, labelField, currentOptionLabel, element)
			if currentOptionValue == selectedValue {
				sel = "selected"
			}
			opts += fmt.Sprintf(`<option value="%d" %s>%s</option>`, currentOptionValue, sel, Escape(currentOptionLabel))
		}

	}
	output := fmt.Sprintf(tmpl, Escape(name), Escape(name), opts)
	return got.HTML(output)
}
func Int64FromString(fromValue string) int64 {
	i64, err := strconv.ParseInt(fromValue, 10, 0)
	if err != nil {
		return 0
	}
	return i64
}
func DynamicScopeValue(scope map[string]interface{}, namePrefix string, nameSuffix int64) int64 {
	namedLookup := namePrefix + strconv.FormatInt(nameSuffix, 10)
	//fmt.Printf("Looking for %s \n", namedLookup)
	for k, v := range scope {
		//fmt.Printf("%s , %s \n", k, v)
		if k == namedLookup {
			result, err := strconv.ParseInt(v.(string), 10, 64)
			if err != nil {
				return 0
			}
			return result
		}
	}
	return 0
}
func DynamicStringValue(scope map[string]interface{}, namePrefix string, nameSuffix int64, value string) bool {
	namedLookup := namePrefix + strconv.FormatInt(nameSuffix, 10)
	for k, v := range scope {
		if k == namedLookup {
			if v.(string) == value {
				return true
			}
		}
	}
	return false
}

func DynamicScopeBooleanValue(scope map[string]interface{}, submapName string, namedLookupInt int64) bool {
	namedLookup := strconv.FormatInt(namedLookupInt, 10)
	for k, value := range scope {
		if k == submapName {
			//fmt.Printf("NamedLookup %s %v\n", namedLookup, value)
			if strings.Contains(value.(string), namedLookup) {
				//fmt.Printf("TRUE\n")
				return true
			}
		}
	}
	return false
}
