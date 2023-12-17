(function() {
    var pageTipClose = document.getElementById('page-flash-close')
    if (pageTipClose) {
        pageTipClose.onclick = function(e) {
            e.preventDefault();
            e.stopPropagation();
            pageTipClose.parentNode.remove();
        }
    }

    addEvents('.btn-reset', 'click', function(ev) {
        ev.preventDefault()
        window.location = ev.target.getAttribute('data-reset-path')
    })
    addEvents('.btn-go-back', 'click', function(ev) {
        ev.preventDefault()
        history.go(-1)
    })

    function addEvents(selector, eventName, fn) {
        document.querySelectorAll(selector).forEach(function(item) {
            item['on' + eventName] = function(ev) {
                if (typeof fn === 'function') {
                    fn.bind(this, ev)()
                }
            }
        })
    }

    const debugUserSel = document.getElementById('debug-user')
    if (debugUserSel) {
        debugUserSel.onchange = function(ev) {
            this.parentNode.submit()
        }
    }

    addEvents('.btn-react', 'change', function(ev) {
        onReactChange(ev, ev.target)
    })

    addEvents('.btn-check-all', 'change', function(ev) {
        checkAllPermission(ev, ev.target)
    })

    addEvents('.btn-check-permission', 'change', function(ev) {
        onPermissionCheck(event, event.target)
    })

    addEvents('form textarea', 'keyup', function(ev) {
        if (ev.isTrusted && ev.ctrlKey && ev.code == 'Enter') {
            var form = ev.target.closest('form')
            if (form) {
                form.submit()
            }
        }
    })

    addEvents('.btn-share', 'click', async function(ev) {
        const shareData = {
            title: '',
            text: '',
            url: ''
        }
        shareData.title = ev.target.getAttribute('data-title')
        shareData.text = ev.target.getAttribute('data-text')
        shareData.url = ev.target.getAttribute('data-url')

        // console.log("share data:", shareData)

        if (navigator.canShare && navigator.canShare(shareData)) {
            ev.preventDefault()
            try {
                await navigator.share(shareData)
            } catch (e) {
                console.error(e)
            }
        }
    })

    // addEvents('.ban-form input[type="radio"]', 'change', function(ev) {
    //     const timeEl = document.querySelector('input.ban-expire')
    //     console.log(ev, timeEl)
    //     console.log(ev.target.value)
    //     const dayNum = parseInt(ev.target.value, 10)
    //     if (dayNum > 0) {
    //         const expTime = new Date(Date.now() + dayNum * 24 * 60 * 60 * 1000)
    //         const [year, month, day, hour, minute] = [expTime.getFullYear(), expTime.getMonth() + 1, expTime.getDate(), expTime.getHours(), expTime.getMinutes()]
    //         let expTimeStr = year
    //         if (month < 10) {
    //             expTimeStr += '-0' + month
    //         } else {
    //             expTimeStr += '-' + month
    //         }

    //         if (day < 10) {
    //             expTimeStr += '-0' + day
    //         } else {
    //             expTimeStr += '-' + day
    //         }

    //         expTimeStr += 'T'

    //         if (hour < 10) {
    //             expTimeStr += '0' + hour
    //         } else {
    //             expTimeStr += '' + hour
    //         }

    //         if (minute < 10) {
    //             expTimeStr += ':0' + minute
    //         } else {
    //             expTimeStr += ':' + minute
    //         }

    //         // console.log("expTimeStr:", expTimeStr)
    //         timeEl.value = expTimeStr
    //     } else {
    //         timeEl.value = ""
    //     }
    // })

    document.querySelectorAll('.input-select-text').forEach(function(el) {
        // console.log('el:', el)
        el.focus()
        el.setSelectionRange(0, el.value.length)
    })

    /*--------------------- article page ---------------------------------------------------*/
    const WIDTH_MOBILE = 750;
    var isMobile = window.innerWidth <= WIDTH_MOBILE

    window.addEventListener('resize', function(e) {
        isMobile = window.innerWidth <= WIDTH_MOBILE
    })

    var replyBox = document.getElementById('replies-box')
    if (replyBox) {
        setUpReplyBox()
    }

    function setUpReplyBox() {
        document.documentElement.addEventListener('mouseup', function(ev) {
            /* console.log(ev) */
            if (replyBox && !replyBox.contains(ev.target)) {
                removeAllActive('li', replyBox)
            }
        })


        var ignoreTags = ['a', 'button', 'select']
        replyBox.addEventListener('mousedown', function(ev) {
            /* console.log('ev', ev) */
            var targetTag = ev.target.tagName.toLowerCase()
            if (ignoreTags.indexOf(targetTag) > -1) return

            var row = ev.target.closest('li')
            if (row) {
                onArticleMouseDown(ev, row)
            }
        })

        var btnParent = document.querySelectorAll('.btn-parent')

        document.querySelectorAll('.btn-parent').forEach(function(item) {
            item.onclick = function(ev) {
                scrollToElementById(ev, item.getAttribute('href').replace(/^#/, ''));
            }
        })

        document.querySelectorAll('.btn-fold').forEach(function(item) {
            item.onclick = function(ev) {
                ev.preventDefault()
                toggleDisplayContent(item.closest('li'), true)
            }
        })

        document.querySelectorAll('.btn-reply-ref').forEach(function(item) {
            item.onclick = function(ev) {
                // console.log("ev target:", ev.target)
                // console.log("ev target data:", ev.target.getAttribute('data-reply-to-id'))
                const replyToId = ev.target.getAttribute('data-reply-to-id')
                const floorId = "ar_" + replyToId
                const floor = document.getElementById(floorId)
                if (floor) {
                    scrollToElementById(ev, floorId)
                }
            }
        })
    }

    function removeAllActive(tagName, parentNode) {
        var parent = parentNode || document
        var els = parent.getElementsByTagName(tagName)
        for (var i = 0; i < els.length; i++) {
            removeClass(els[i], 'active')
        }
    }

    // return previous display or not
    function toggleDisplayEl(el, toDisplay) {
        var flag = el.style.display == 'none'
        if (typeof toDisplay == 'boolean') {
            flag = toDisplay
        }

        if (flag) {
            el.style.display = ''
            return false
        } else {
            el.style.display = 'none'
            return true
        }
    }

    function toggleDisplayContent(rowEl, toFold) {
        var content = rowEl.getElementsByTagName('section')[0]
        var operation = rowEl.getElementsByClassName('article-operation')[0]
        var id = rowEl.getAttribute('data-id')

        if (content) {
            var isDisplay = checkDisplay(content)
            var placeholderId = 'article_placeholder_' + id
            var placeholder

            var replyNumText = rowEl.getAttribute('data-reply-num-self-include-text')

            var flag = isDisplay
            if (typeof toFold == 'boolean') {
                flag = toFold
            }

            toggleDisplayEl(content, !flag)
            toggleDisplayEl(operation, !flag)

            if (flag) {
                placeholder = document.createElement('div')
                placeholder.setAttribute('id', placeholderId)
                placeholder.className = 'article-placeholder text-lighten-2'
                placeholder.innerHTML = '&nbsp;&nbsp;<i>' + replyNumText + '</i>'
                content.parentNode.insertBefore(placeholder, content)

                rowEl.setAttribute('data-hidden', 1)
            } else {
                placeholder = document.getElementById(placeholderId)
                if (placeholder) placeholder.remove()

                rowEl.removeAttribute('data-hidden')
            }
        }

        var child = rowEl.getElementsByTagName('ul')[0]
        if (child) {
            toggleDisplayEl(child, !flag)
        }

        /* console.log('isMobile', isMobile) */

        if (isMobile && toFold) {
            window.scrollTo({
                top: rowEl.getBoundingClientRect().top + document.documentElement.scrollTop,
                behavior: 'smooth'
            })
        }
    }

    function toggleActive(el) {
        var isActive = hasClass(el, 'active')
        var isHidden = !!el.getAttribute("data-hidden")

        removeAllActive('li', document.getElementById('replies-box'))
        addClass(el, 'active')

        if (isHidden) {
            toggleDisplayContent(el)
        }
    }

    function checkDisplay(el) {
        return el.style.display != 'none'
    }

    function hasClass(el, className) {
        return el.className.split(" ").indexOf(className) > -1
    }

    function removeClass(el, className) {
        if (hasClass(el, className)) {
            el.className = el.className.replace(className, '')
        }
    }

    function addClass(el, className) {
        if (!hasClass(el, className)) {
            el.className += " " + className
        }

    }

    function onArticleMouseDown(ev, rowEl) {
        function onArticleMouseUp() {
            /* console.log(ev, el) */
            event.preventDefault()
            event.stopPropagation()

            toggleActive(rowEl)

            // if (hasClass(rowEl, 'active')){
            // toggleDisplayContent(rowEl)
            // } else {
            // toggleActive(rowEl)
            // }
        }

        rowEl.onmouseup = onArticleMouseUp
        setTimeout(function() {
            rowEl.onmouseup = null
        }, 300)
    }

    function scrollToElementById(ev, id) {
        ev.preventDefault()
        ev.stopPropagation()
        var el = document.getElementById(id)
        /* console.log(ev, el) */
        if (el) {
            setTimeout(function() {
                location.hash = id
            }, 500)
            window.scrollTo({
                top: el.getBoundingClientRect().top + document.documentElement.scrollTop,
                behavior: 'smooth',
            })
        }
    }

    function onReactChange(ev, sel) {
        sel.closest('form').submit()
        setTimeout(function() {
            sel.value = ""
        }, 0)
    }

    /*--------------------- role edit page  ---------------------------------------------------*/
    document.querySelectorAll('.btn-all').forEach(el => toggleBtnAll(el))

    function checkAllPermission(ev, el) {
        /* console.log(event, el) */
        el.closest('fieldset').querySelectorAll('input[name="permissions"]').forEach(item => {
            /* console.log("item: ", item) */
            item.checked = el.checked
        })
    }

    function onPermissionCheck(ev, el) {
        /* console.log(ev, el) */
        toggleBtnAll(el)
    }

    function toggleBtnAll(el) {
        const allBtn = el.closest('fieldset').querySelector('input.btn-check-all')
        const inputEls = el.closest('fieldset').querySelectorAll('input[name="permissions"]')
        let allChecked = true

        inputEls.forEach(item => {
            /* console.log(item, item.checked) */
            if (!item.checked) {
                allChecked = false
            }
        })

        allBtn.checked = allChecked
    }
})()
